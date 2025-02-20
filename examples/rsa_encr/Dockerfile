FROM python:3.10-slim
RUN pip3 install cryptography jsonschema

COPY rsa_executor.py /
COPY rsa_func.py /

WORKDIR /
CMD python rsa_executor.py
