"""Qwen/ComfyUI image-edit bridge for CAWForge face textures.

Requires a configured API-format workflow, two LoadImage nodes, a positive
prompt node, and a SaveImage node. No model installation is performed.
"""
import argparse
import json
from pathlib import Path
import time
import uuid
import urllib.error
import urllib.parse
import urllib.request
from io import BytesIO


def request_json(url, payload=None, timeout=30):
    data = json.dumps(payload).encode("utf-8") if payload is not None else None
    req = urllib.request.Request(url, data=data,
                                 headers={"Content-Type": "application/json"} if data is not None else {})
    with urllib.request.urlopen(req, timeout=timeout) as response:
        return json.load(response)


def upload_image(server, path):
    path = Path(path)
    if not path.is_file():
        raise FileNotFoundError(path)
    boundary = "cawforge-" + uuid.uuid4().hex
    filename = path.name.replace('"', "_").replace("\r", "_").replace("\n", "_")
    head = ('--' + boundary + '\r\nContent-Disposition: form-data; '
            'name="image"; filename="' + filename + '"\r\n'
            'Content-Type: image/png\r\n\r\n').encode()
    tail = ('\r\n--' + boundary + '--\r\n').encode()
    payload = head + path.read_bytes() + tail
    req = urllib.request.Request(server + "/upload/image", data=payload,
                                 headers={"Content-Type": "multipart/form-data; boundary=" + boundary})
    with urllib.request.urlopen(req, timeout=90) as response:
        obj = json.load(response)
    if not obj.get("name"):
        raise RuntimeError("ComfyUI did not return an uploaded image name")
    return obj["name"]


def configured_workflow(workflow, template_node, photo_node, prompt_node,
                        template_name, photo_name, prompt):
    if not isinstance(workflow, dict) or not workflow:
        raise ValueError("Workflow must be a nonempty API-format node dictionary")
    for node_id in (template_node, photo_node, prompt_node):
        if node_id not in workflow or not isinstance(workflow[node_id].get("inputs"), dict):
            raise ValueError("Missing workflow node or inputs: " + node_id)
    if template_node == photo_node:
        raise ValueError("Template and photo must use different LoadImage nodes")
    for node_id in (template_node, photo_node):
        if workflow[node_id].get("class_type") != "LoadImage":
            raise ValueError("Image input node " + node_id + " must be LoadImage")
    if not any(n.get("class_type") == "SaveImage" for n in workflow.values()):
        raise ValueError("API workflow must contain a SaveImage node")
    if not isinstance(workflow[prompt_node]["inputs"].get("text"), str):
        raise ValueError("Prompt node requires literal string text input")
    workflow[template_node]["inputs"]["image"] = template_name
    workflow[photo_node]["inputs"]["image"] = photo_name
    workflow[prompt_node]["inputs"]["text"] = prompt
    return workflow


def save_texture(raw, destination):
    from PIL import Image
    with Image.open(BytesIO(raw)) as original:
        original.load()
        if original.width != original.height:
            raise ValueError("Qwen generated a non-square image; refusing to crop UV landmarks")
        if min(original.size) < 512:
            raise ValueError("Qwen generated an image too small for a usable face texture")
        img = original.convert("RGBA")
        if img.size != (820, 820):
            img = img.resize((820, 820), Image.Resampling.LANCZOS)
        destination = Path(destination)
        destination.parent.mkdir(parents=True, exist_ok=True)
        img.save(destination, "PNG")
    return destination


def generate(args):
    parsed = urllib.parse.urlsplit(args.server)
    if parsed.scheme != "http" or parsed.hostname not in ("localhost", "127.0.0.1", "::1"):
        raise ValueError("For safety, the ComfyUI server must be local HTTP (127.0.0.1)")
    server = args.server.rstrip("/")
    workflow = json.loads(Path(args.workflow).read_text(encoding="utf-8"))
    # Validate configuration before uploading.
    configured_workflow(json.loads(json.dumps(workflow)), args.input_node, args.reference_node,
                        args.prompt_node, "template.png", "photo.png", args.prompt)
    template_name = upload_image(server, args.input)
    photo_name = upload_image(server, args.reference)
    workflow = configured_workflow(workflow, args.input_node, args.reference_node,
                                   args.prompt_node, template_name, photo_name, args.prompt)
    client_id = uuid.uuid4().hex
    queued = request_json(server + "/prompt", {"prompt": workflow, "client_id": client_id})
    prompt_id = queued.get("prompt_id")
    if not prompt_id:
        raise RuntimeError("ComfyUI rejected workflow: " + json.dumps(queued)[:2000])
    deadline = time.monotonic() + args.timeout
    while time.monotonic() < deadline:
        history = request_json(server + "/history/" + urllib.parse.quote(str(prompt_id), safe=""))
        result = history.get(str(prompt_id), {})
        if result:
            status = result.get("status", {})
            if status.get("status_str") == "error":
                raise RuntimeError("ComfyUI execution error: " + json.dumps(status.get("messages", []))[:2000])
            # Use only SaveImage outputs, avoiding unrelated previews.
            for node_id, node in workflow.items():
                if node.get("class_type") != "SaveImage":
                    continue
                images = result.get("outputs", {}).get(node_id, {}).get("images", [])
                if images:
                    descriptor = images[0]
                    query = urllib.parse.urlencode({
                        "filename": descriptor["filename"],
                        "subfolder": descriptor.get("subfolder", ""),
                        "type": descriptor.get("type", "output"),
                    })
                    with urllib.request.urlopen(server + "/view?" + query, timeout=120) as response:
                        return save_texture(response.read(), args.output)
            if status.get("completed"):
                raise RuntimeError("Workflow finished without a SaveImage output")
        time.sleep(2)
    raise TimeoutError("ComfyUI exceeded the specified generation timeout")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workflow", required=True)
    parser.add_argument("--input", required=True)
    parser.add_argument("--reference", required=True)
    parser.add_argument("--input-node", required=True)
    parser.add_argument("--reference-node", required=True)
    parser.add_argument("--prompt-node", required=True)
    parser.add_argument("--prompt", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--server", default="http://127.0.0.1:8188")
    parser.add_argument("--timeout", type=int, default=1200)
    args = parser.parse_args(argv)
    if not 1 <= args.timeout <= 7200:
        parser.error("--timeout must be between 1 and 7200 seconds")
    try:
        path = generate(args)
        print("Qwen face texture saved:", path)
    except (OSError, ValueError, RuntimeError, TimeoutError, KeyError, json.JSONDecodeError) as exc:
        parser.exit(1, "Qwen backend error: " + str(exc) + "\n")


if __name__ == "__main__":
    main()
