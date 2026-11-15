"""CAWForge Qwen readiness check (no downloads, no model installation).

Run beside CAWForge.exe: python qwen_setup_check.py
"""
import json
import sys
import urllib.request
from pathlib import Path

ROOT=Path(__file__).resolve().parent
CONFIG=ROOT/"qwen_face_config.json"
EXAMPLE=ROOT/"qwen_face_config.example.json"

def main():
    errors=[]
    print("CAWForge Qwen Face Texture setup check")
    if not CONFIG.exists():
        errors.append("Missing qwen_face_config.json. Copy qwen_face_config.example.json and set the workflow/node IDs.")
        cfg={}
    else:
        try:
            cfg=json.loads(CONFIG.read_text(encoding="utf-8-sig"))
        except (ValueError,OSError) as exc:
            errors.append("Invalid config JSON: "+str(exc))
            cfg={}
    for field in ("workflow","template_node","photo_node","prompt_node"):
        value=cfg.get(field,"")
        if not isinstance(value,str) or not value or value.startswith("REPLACE_"):
            errors.append("Configure "+field+" in qwen_face_config.json")
    workflow=cfg.get("workflow","")
    if workflow:
        path=Path(workflow)
        if not path.is_absolute():
            path=ROOT/path
        if not path.is_file():
            errors.append("Missing ComfyUI API workflow: "+str(path))
        else:
            try:
                graph=json.loads(path.read_text(encoding="utf-8-sig"))
                if not isinstance(graph,dict):
                    errors.append("Workflow must be exported in ComfyUI API format, not UI format")
                else:
                    for key,cls in (("template_node","LoadImage"),("photo_node","LoadImage")):
                        node=graph.get(cfg.get(key,""),{})
                        if node.get("class_type")!=cls:
                            errors.append(key+" must point to a LoadImage node")
                    node=graph.get(cfg.get("prompt_node",""),{})
                    if not isinstance(node.get("inputs",{}).get("text"),str):
                        errors.append("prompt_node must have a literal text input")
                    if not any(n.get("class_type")=="SaveImage" for n in graph.values()):
                        errors.append("Workflow needs a SaveImage node")
            except (ValueError,OSError) as exc:
                errors.append("Cannot read workflow: "+str(exc))
    server=cfg.get("server") or "http://127.0.0.1:8188"
    if not (server.startswith("http://127.0.0.1:") or server.startswith("http://localhost:")):
        errors.append("Only local ComfyUI servers are supported for face photos")
    else:
        try:
            with urllib.request.urlopen(server.rstrip("/")+"/system_stats",timeout=5) as response:
                info=json.load(response)
            print("ComfyUI reachable; devices:",json.dumps(info.get("devices",[]))[:500])
        except (OSError,ValueError) as exc:
            errors.append("ComfyUI is not running/reachable at "+server+": "+str(exc))
    if errors:
        print("\nNOT READY:")
        for issue in errors: print(" -",issue)
        print("\nStable Diffusion Render Creator is unaffected.")
        return 1
    print("\nConnection and workflow configuration ready for a generation test.")
    print("This does NOT verify that Qwen model weights are installed or fit GPU memory.")
    return 0

if __name__=="__main__":
    sys.exit(main())
