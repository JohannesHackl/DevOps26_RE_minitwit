FROM golang:alpine3.23

WORKDIR /usr/src/app

ENV DB_IP="dbserver"

COPY requirements.txt ./
##RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD [ "go", "./minitwit.go" ]

