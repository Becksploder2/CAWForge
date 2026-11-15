"""Offline checks of CAWForge Qwen workflow wiring and UV export."""
import json
from io import BytesIO
from pathlib import Path
import tempfile
import unittest

from qwen_backend import configured_workflow, save_texture


def workflow():
    return {
        "1": {"class_type": "LoadImage", "inputs": {"image": "old.png"}},
        "2": {"class_type": "LoadImage", "inputs": {"image": "old2.png"}},
        "3": {"class_type": "CLIPTextEncode", "inputs": {"text": "old"}},
        "4": {"class_type": "SaveImage", "inputs": {"images": ["9", 0]}}
    }


class QwenBackendTests(unittest.TestCase):
    def test_two_inputs_and_prompt_are_assigned(self):
        result = configured_workflow(workflow(), "1", "2", "3", "template.png", "photo.png", "details")
        self.assertEqual(result["1"]["inputs"]["image"], "template.png")
        self.assertEqual(result["2"]["inputs"]["image"], "photo.png")
        self.assertEqual(result["3"]["inputs"]["text"], "details")

    def test_same_node_is_rejected(self):
        with self.assertRaises(ValueError):
            configured_workflow(workflow(), "1", "1", "3", "t", "p", "prompt")

    def test_wrong_class_type_is_rejected(self):
        with self.assertRaises(ValueError):
            configured_workflow(workflow(), "3", "2", "1", "t", "p", "prompt")

    def test_missing_save_image_is_rejected(self):
        nodes = workflow()
        del nodes["4"]
        with self.assertRaises(ValueError):
            configured_workflow(nodes, "1", "2", "3", "t", "p", "prompt")

    def test_output_is_820_square_and_png(self):
        from PIL import Image
        source = BytesIO()
        Image.new("RGB", (1024, 1024), (161, 115, 95)).save(source, "PNG")
        with tempfile.TemporaryDirectory() as temp:
            target = save_texture(source.getvalue(), Path(temp) / "face.png")
            with Image.open(target) as result:
                self.assertEqual(result.size, (820, 820))
                self.assertEqual(result.format, "PNG")

    def test_rectangular_output_cannot_be_cropped(self):
        from PIL import Image
        source = BytesIO()
        Image.new("RGB", (1024, 768)).save(source, "PNG")
        with tempfile.TemporaryDirectory() as temp:
            with self.assertRaisesRegex(ValueError, "non-square"):
                save_texture(source.getvalue(), Path(temp) / "face.png")


if __name__ == "__main__":
    unittest.main()
