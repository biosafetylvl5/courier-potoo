import os
import time
import random
import string
from pathlib import Path
from loguru import logger
from PIL import Image
import argparse
import numpy as np
import threading

IMG_HEIGHT = 1024
IMG_WIDTH = 1024

def init_argparse() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="courier-potoo",
        description="helper demo for courier",
    )
    # collection of images to be converted to
    parser.add_argument("imagepath", type=str)

    # time between generation of random images
    parser.add_argument("--timeout", "-t", type=int, default=0.5)
    # where the random images are stored
    parser.add_argument("--outputpath", "-o", type=str, default=".")
    # filetype
    parser.add_argument("--filetype", type=str, default="png")

    return parser.parse_args()


class PotooWriter:
    def __init__(self,
                 args: argparse.Namespace) -> None:
        self._color_scheme_len : int = 5
        self._image_path : Path = Path(args.imagepath)
        self._output_path : Path = Path(args.outputpath)
        self._filetype : str = args.filetype
        self._timeout : int = args.timeout
    def _generate_tag(self, length=8):
        characters = string.ascii_letters + string.digits

        tag = ''.join(random.choices(characters, k=length))
        # e.g. keyboard smash
        return tag
    def _generate_random_image(self) -> Image.Image:
        color_scheme = np.array([
            tuple(random.randint(0,255) for _ in range(3))
            for _ in range(self._color_scheme_len)
        ], dtype=np.uint8)

        random_image = color_scheme[
            np.random.randint(0, len(color_scheme), size=(IMG_HEIGHT, IMG_WIDTH))
        ]

        return Image.fromarray( random_image )
    def _save_image(self, image : Image.Image) -> Path:
        tag = self._generate_tag()
        path_name = f"{os.path.join(self._output_path, tag)}.{self._filetype}"
        image.save(path_name)
        return Path(path_name)
        
    def drive_writer(self) -> None:
        while(True):
            generated_image = self._generate_random_image()
            saved_image = self._save_image(generated_image)
            logger.debug(f"Saved random image to {saved_image}")
            time.sleep(self._timeout)

if __name__ == "__main__":
    args = init_argparse()
    writer = PotooWriter(args)
    writer_thread = threading.Thread(
        target=writer.drive_writer)
    writer_thread.start()

    writer_thread.join()




