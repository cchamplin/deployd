[Unit]
Description={{.service_id}} php container

[Service]
ExecStart=/usr/bin/docker run --rm --name php_{{.service_id}} php-image
ExecStartPost=/opt/bin/waitcheck /opt/containers/{{.service_id}}/run/php.sock
ExecStartPost=/opt/bin/set-server-info.sh {{.service_id}}


ExecStop=/usr/bin/docker stop php_{{.service_id}}
