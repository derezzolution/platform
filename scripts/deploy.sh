#!/bin/bash -e
# Deploys the platform

SERVICE_NAME=platform
cd $(dirname $0)/..

if [ "$EUID" -ne 0 ]; then
    echo "please run deploy as root"
    exit
fi

echo "Stop $SERVICE_NAME and uninstall"
(
    set -o xtrace

    # Stop service
    systemctl stop $SERVICE_NAME || true
    systemctl disable $SERVICE_NAME || true

    # Uninstall
    rm /etc/systemd/system/$SERVICE_NAME.service || true
    rm -rf /opt/$SERVICE_NAME || true
)

echo "Install and start $SERVICE_NAME"
(
    set -o xtrace

    # Install
    mkdir /opt/$SERVICE_NAME
    mv config-production.json /opt/$SERVICE_NAME/
    mv deployer /opt/$SERVICE_NAME/
    chown root: -R /opt/$SERVICE_NAME
    chmod g=,o= -R /opt/$SERVICE_NAME

    # Write the service file
    cat <<EOF >/etc/systemd/system/$SERVICE_NAME.service
# Make sure to enable this Unit so it comes up on reboot with
# \`systemctl enable $SERVICE_NAME\`

[Unit]
Description=$SERVICE_NAME
After=network.target

[Service]
Environment=GO_ENV=production
TimeoutStartSec=0
WorkingDirectory=/opt/$SERVICE_NAME
ExecStart=/opt/$SERVICE_NAME/$SERVICE_NAME -hidetimestamp
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
EOF

    # Start deployer
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME
)
