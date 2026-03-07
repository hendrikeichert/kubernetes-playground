FROM python:slim
  WORKDIR /app
  COPY requirements.txt .
  RUN pip install --no-cache-dir -r requirements.txt
  COPY app.py .
  COPY data /app/data
  EXPOSE 9080
  CMD ["python", "app.py"]