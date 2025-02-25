FROM python:3.10-slim
RUN pip3 install Pillow jsonschema

COPY executor.py /
COPY function.py /
COPY ops.py /

WORKDIR /
CMD python executor.py