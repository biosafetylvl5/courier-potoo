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
    # does this program write random files? if not, it's an ingester
    parser.add_argument("--write", "-w", type=bool, default=False)
    # collection of images to be converted to
    parser.add_argument("--imagepath", "-I", type=str, default="")

    # time between generation of random images
    parser.add_argument("--timeout", "-t", type=int, default=0.5)
    # where the random images are stored
    parser.add_argument("--outputpath", "-o", type=str, default=".")
    # filetype
    parser.add_argument("--filetype", type=str, default="png")

    # image ingest
    parser.add_argument("--input", "-i", type=str, default=None)
    # how many steps to get to target image?
    parser.add_argument("--iterations", "-it", type=int, default=25)
    # duration of generated gif
    parser.add_argument("--duration", "-d", type=int, default=100)

    return parser.parse_args()


class Potoo:
    def __init__(self,
                 args: argparse.Namespace) -> None:
        self._color_scheme_len : int = 5
        self._image_path : Path = Path(args.imagepath)
        self._output_path : Path = Path(args.outputpath)
        self._filetype : str = args.filetype
        self._timeout : int = args.timeout
        self._iterations = args.iterations
        self._duration = args.duration
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
    def _generate_gif(self, steps_arr : list[Image.Image], tag : str) -> Path:
        output_path = f"{os.path.join(self._output_path, tag)}.gif"
        steps_arr[0].save(output_path, save_all=True, 
                          append_images=steps_arr, duration=self._duration)
        return Path(output_path)
    def ingest_image(self, source : Path) -> None:
        # get random image from imagepath
        source_img = Image.open(source)
        source_img = source_img.resize((IMG_HEIGHT, IMG_WIDTH), Image.Resampling.LANCZOS)
        source_arr = np.asarray(source_img, copy=True)
        target = Image.open(random.choice(
            [f for f in self._image_path.iterdir() if f.is_file()]))
        target = target.resize((IMG_HEIGHT, IMG_WIDTH), Image.Resampling.LANCZOS)
        target_arr = np.asarray(target)

        step = ((target_arr - source_arr) / self._iterations).astype(np.uint8)
        steps_arr = []

        for _ in range(self._iterations):
            np.add(source_arr, step, out=source_arr)
            steps_arr.append(Image.fromarray( source_arr ))
        generated_path = self._generate_gif(steps_arr, source.stem)
        logger.debug(f"Saved GIF to {generated_path}")
if __name__ == "__main__":
    args = init_argparse()
    potoo = Potoo(args)
    if args.write:
        potoo_thread = threading.Thread(
            target=potoo.drive_writer)
        potoo_thread.start()
        potoo_thread.join()
    else:
        if not args.input or not args.imagepath:
            raise ValueError("Image ingest required for non-writer.")
        input = Path(args.input)
        potoo.ingest_image(input)
