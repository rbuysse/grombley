FROM nginx:latest

COPY tests/scripts/* /tmp/

COPY tests/images/* /tmp/
COPY tests/images/test.jpg /tmp/test.png
COPY tests/images/* /usr/share/nginx/html/
COPY tests/images/test.jpg /usr/share/nginx/html/jpg.not.zip
COPY tests/images/test.jpg /usr/share/nginx/html/extlessjpg
