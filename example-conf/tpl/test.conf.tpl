{{range (getvs "deployments/sites")}}
server {
    listen 80;
    server_name "{{.sitehost}}";
    root /mnt/{{.sitename}}/www;
    index index.php index.html;

    sendfile off;
	  location / {
      try_files $uri $uri/ @rewrite;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico)$ {
       expires max;
       log_not_found off;
       access_log off;
       add_header ETag "";
    }
	  location @rewrite {
      rewrite / /index.php?$args;
    }
    location ~ \.php$ {
      try_files $uri =404;
      # fastcgi_split_path_info ^(.+\.php)(/.+)$;
      fastcgi_pass {{.sitename}};
      fastcgi_param SCRIPT_FILENAME /mnt/www$fastcgi_script_name;
      fastcgi_param SCRIPT_NAME $fastcgi_script_name;
      fastcgi_index index.php;
      include fastcgi_params;
	    fastcgi_connect_timeout 65;
	    fastcgi_send_timeout 7200;
      fastcgi_read_timeout 7200;
    }
    location /favicon.ico {
        log_not_found off;
    }
}

upstream {{.sitename}} {
  server unix:/mnt/run/{{.sitename}}-php.sock;
}
{{end}}
