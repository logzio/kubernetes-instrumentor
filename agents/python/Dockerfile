FROM python:3.11-alpine AS build

ADD requirements.txt .
RUN mkdir autoinstrumentation && pip install --target autoinstrumentation -r requirements.txt

RUN chmod -R go+r /autoinstrumentation