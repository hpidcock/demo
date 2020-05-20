#!/bin/sh
set -ex

wget -P style_image/ -v --content-disposition "${STYLE_IMAGE}"
wget -P content_image/ -v --content-disposition "${CONTENT_IMAGE}"
neural-style -model_file /models/vgg19-* -style_image style_image/* -content_image content_image/* -output_image output_image.png -backend cudnn -optimizer adam -image_size 512 -gpu 0
curl -F file=@output_image.png -v "${UPLOAD_URL}"
