FROM nginx:latest

COPY tests/scripts/* /tmp/

COPY tests/images/* /tmp/
COPY tests/test.mp3 /tmp/test.mp3
COPY tests/images/* /usr/share/nginx/html/
COPY tests/images/test.jpg /usr/share/nginx/html/jpg.not.zip
COPY tests/images/test.jpg /usr/share/nginx/html/extlessjpg
