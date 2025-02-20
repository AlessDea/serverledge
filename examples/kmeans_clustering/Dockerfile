FROM python:3.10-slim
RUN pip3 install scikit-learn numpy jsonschema

COPY km_executor.py /
COPY function.py /

WORKDIR /
CMD python km_executor.py