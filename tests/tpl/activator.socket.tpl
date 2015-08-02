[Socket]
ListenStream=/opt/containers/run/php-{{.service_id}}.sock

[Install]
WantedBy=sockets.target
