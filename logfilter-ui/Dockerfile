FROM alpine

RUN apk add apache2 apache2-utils python wget \
 && wget https://storage.googleapis.com/kubernetes-release/release/v1.11.2/bin/linux/amd64/kubectl \
 && mv kubectl /usr/bin \
 && chmod 755 /usr/bin/kubectl

RUN mkdir /run/apache2 \
 && chown apache /run/apache2 \
 && sed -i 's/#LoadModule cgid_module/LoadModule cgid_module/g' /etc/apache2/httpd.conf \
 && sed -i 's/#LoadModule cgi_module/LoadModule cgi_module/g' /etc/apache2/httpd.conf \
 && sed -i 's/Options Indexes FollowSymLinks/Options Indexes FollowSymLinks ExecCGI/g' /etc/apache2/httpd.conf \
 && sed -i 's/DirectoryIndex index.html/DirectoryIndex index.py/g' /etc/apache2/httpd.conf \
 && sed -i 's/AllowOverride None/AllowOverride All/g' /etc/apache2/httpd.conf \
 && echo "AddHandler cgi-script .cgi .py" >> /etc/apache2/httpd.conf

COPY htdocs /var/www/localhost/htdocs
COPY htaccess /var/www/localhost/htdocs/.htaccess
COPY entrypoint.sh /
RUN chmod 755 /var/www/localhost/htdocs/index.py \
 && chmod 755 /entrypoint.sh

CMD ["/entrypoint.sh"]
