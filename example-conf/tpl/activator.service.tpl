[Unit]
Requires=php-{{.service_id}}.service
After=php-{{.service_id}}.service

[Service]
ExecStart=/opt/bin/launcher.sh {{.service_id}}
