import uuid
from time import time
from PIL import Image
import ops
import os
import base64

FILE_NAME_INDEX = -1

def image_processing(file_name, image_path):
    path_list = []
    start = time()

    with Image.open(image_path) as image:
        path_list += ops.flip(image, file_name)
        path_list += ops.rotate(image, file_name)
        path_list += ops.filter(image, file_name)
        path_list += ops.gray_scale(image, file_name)
        path_list += ops.resize(image, file_name)

    latency = time() - start
    return latency, path_list

def handler(params, context):
    try:
        file_name = params["file_name"]
        file_content = params["file_content"]

        file_content = base64.b64decode(file_content)

        temp_dir = "/tmp/"
        image_path = os.path.join(temp_dir, f"{uuid.uuid4()}_{file_name}")

        with open(image_path, "wb") as f:
            f.write(file_content)

        latency, path_list = image_processing(file_name, image_path)

        return {
            "latency": latency,
            "processed_files": path_list
        }

    except Exception as e:
        return {"error": str(e)}
