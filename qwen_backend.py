"""Experimental Qwen-Image-Edit bridge for CAWForge.

Requires a separately installed ComfyUI with a working Qwen Image Edit workflow.
This module intentionally does NOT download large models or modify the existing
Stable Diffusion render pipeline.

Usage:
 python qwen_backend.py --check
 python qwen_backend.py --workflow qwen_edit_api.json --input source.png --output result.png --prompt "..." 

The workflow must be exported from ComfyUI in API format and contain one
LoadImage node, one SaveImage node, and a prompt text node selected using
--prompt-node. Model and node compatibility depend on the installed workflow.
"""
import argparse
import json
import time
import urllib.request
import urllib.error
import uuid
from pathlib import Path

def request_json(url, payload=None, timeout=20):
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers={"Content-Type":"application/json"} if data else {})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.load(resp)

def upload_image(base, path):
    boundary = "----cawforge" + uuid.uuid4().hex
    raw = Path(path).read_bytes()
    name = Path(path).name
    body = (
        f"--{boundary}\r\nContent-Disposition: form-data; name=\"image\"; filename=\"{name}\"\r\n"
        "Content-Type: image/png\r\n\r\n"
    ).encode() + raw + f"\r\n--{boundary}--\r\n".encode()
    req = urllib.request.Request(base + "/upload/image", data=body, headers={"Content-Type":f"multipart/form-data; boundary={boundary}"})
    with urllib.request.urlopen(req, timeout=90) as resp:
        return json.load(resp)["name"]

def main():
    p=argparse.ArgumentParser()
    p.add_argument("--server",default="http://127.0.0.1:8188")
    p.add_argument("--check",action="store_true")
    p.add_argument("--workflow")
    p.add_argument("--input")
    p.add_argument("--output")
    p.add_argument("--prompt",default="")
    p.add_argument("--prompt-node",default="")
    p.add_argument("--timeout",type=int,default=1200)
    a=p.parse_args()
    base=a.server.rstrip("/")
    try:
        info=request_json(base+"/system_stats")
    except (OSError,ValueError) as exc:
        raise SystemExit("ComfyUI not reachable at "+base+": "+str(exc))
    if a.check:
        print("ComfyUI reachable:",base)
        print("Devices:",json.dumps(info.get("devices",[]))[:1000])
        print("Qwen model availability NOT verified; load and test an API workflow first.")
        return
    if not a.workflow or not a.input or not a.output:
        p.error("--workflow, --input and --output required")
    graph=json.loads(Path(a.workflow).read_text(encoding="utf-8"))
    if not isinstance(graph,dict) or not graph:
        raise SystemExit("Expected a ComfyUI API-format workflow JSON object")
    image_nodes=[n for n in graph.values() if n.get("class_type")=="LoadImage"]
    save_nodes=[n for n in graph.values() if n.get("class_type")=="SaveImage"]
    if len(image_nodes)!=1 or len(save_nodes)!=1:
        raise SystemExit("Workflow must have exactly one LoadImage and one SaveImage node")
    image_nodes[0]["inputs"]["image"]=upload_image(base,a.input)
    if a.prompt_node:
        if a.prompt_node not in graph or "text" not in graph[a.prompt_node].get("inputs",{}):
            raise SystemExit("--prompt-node must identify a workflow node with a text input")
        graph[a.prompt_node]["inputs"]["text"]=a.prompt
    elif a.prompt:
        raise SystemExit("Specify --prompt-node to safely inject text into the correct workflow node")
    client=uuid.uuid4().hex
    queued=request_json(base+"/prompt",{"prompt":graph,"client_id":client})
    pid=queued["prompt_id"]
    deadline=time.monotonic()+a.timeout
    while time.monotonic()<deadline:
        history=request_json(base+"/history/"+pid)
        if pid in history:
            run=history[pid]
            if run.get("status",{}).get("status_str")=="error":
                raise SystemExit("ComfyUI workflow failed: "+json.dumps(run.get("status",{}))[:2000])
            for node in run.get("outputs",{}).values():
                for img in node.get("images",[]):
                    if img.get("type")=="output":
                        from urllib.parse import urlencode
                        url=base+"/view?"+urlencode({"filename":img["filename"],"subfolder":img.get("subfolder",""),"type":"output"})
                        with urllib.request.urlopen(url,timeout=90) as resp:
                            Path(a.output).parent.mkdir(parents=True,exist_ok=True)
                            Path(a.output).write_bytes(resp.read())
                        print("Qwen experimental output:",a.output)
                        return
            raise SystemExit("Workflow finished without a saved output image")
        time.sleep(3)
    raise SystemExit("Timed out waiting for ComfyUI. Model may be too large for this hardware.")

if __name__=="__main__":
    main()
