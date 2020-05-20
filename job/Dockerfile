FROM nvidia/cuda:10.2-cudnn7-runtime

RUN apt-get update && apt-get install -y software-properties-common && add-apt-repository ppa:deadsnakes/ppa && apt-get update && apt-get install -y python3.8 python3.8-distutils wget curl
RUN curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
RUN python3.8 get-pip.py
RUN python3.8 -m pip install neural-style torch==1.4.0+cu100 torchvision==0.5.0+cu100 -f https://download.pytorch.org/whl/torch_stable.html

RUN mkdir /models
RUN neural-style -download_models /models

ADD script.sh /usr/local/bin/script.sh
RUN chmod +x /usr/local/bin/script.sh

ENTRYPOINT ["/usr/local/bin/script.sh"]
