#!/bin/env bash
set -e

R_PATH=/usr/bin/R
RSERVER_PATH=/usr/lib/rstudio-server/bin/rserver

# store coder agent variables for rstudio session
mkdir -p "$HOME/.config/rstudio"
printenv | grep '^CODER_\|^GIT_\|^SSH_' > "$HOME/.config/rstudio/coder.env"

if [ ! -f "$R_PATH" ]; then
    # install R with r2u packages (https://github.com/eddelbuettel/r2u)
    # taken from https://github.com/eddelbuettel/r2u/blob/master/inst/scripts/add_cranapt_noble.sh
    sudo sh -c 'apt update -qq && apt install -y --no-install-recommends ca-certificates gnupg'
    sudo sh -c 'wget -q -O- https://eddelbuettel.github.io/r2u/assets/dirk_eddelbuettel_key.asc > /etc/apt/trusted.gpg.d/cranapt_key.asc'
    sudo sh -c 'wget -q -O- https://cloud.r-project.org/bin/linux/ubuntu/marutter_pubkey.asc > /etc/apt/trusted.gpg.d/cran_ubuntu_key.asc'
    sudo sh -c 'gpg --keyserver keyserver.ubuntu.com --recv-keys 67C2D66C4B1D4339 51716619E084DAB9'
    sudo sh -c 'gpg --export --armor 67C2D66C4B1D4339 51716619E084DAB9 > /usr/share/keyrings/r2u.gpg'
    sudo sh -c 'echo "deb [arch=amd64] https://r2u.stat.illinois.edu/ubuntu noble main" > /etc/apt/sources.list.d/cranapt.list'
    sudo sh -c 'echo "deb [arch=amd64] https://cloud.r-project.org/bin/linux/ubuntu noble-cran40/" > /etc/apt/sources.list.d/cran_r.list'
    sudo sh -c 'apt update -qq && apt install -y --no-install-recommends r-base-core'
    
    # add pinning to ensure package sorting
    sudo sh -c 'echo "Package: *" > /etc/apt/preferences.d/99cranapt'
    sudo sh -c 'echo "Pin: release o=CRAN-Apt Project" >> /etc/apt/preferences.d/99cranapt'
    sudo sh -c 'echo "Pin: release l=CRAN-Apt Packages" >> /etc/apt/preferences.d/99cranapt'
    sudo sh -c 'echo "Pin-Priority: 700"  >> /etc/apt/preferences.d/99cranapt'

    # install bsm
    sudo sh -c 'apt install -y --no-install-recommends python3-dbus python3-gi python3-apt make'
    sudo sh -c "Rscript -e 'install.packages(\"bspm\")'"
    sudo sh -c 'echo "suppressMessages(bspm::enable())" >> /usr/lib/R/etc/Rprofile.site'
    sudo sh -c 'echo "options(bspm.version.check=FALSE)" >> /usr/lib/R/etc/Rprofile.site'
fi

if [ ! -f "$RSERVER_PATH" ]; then
    # install rstudio server
    DEB=/tmp/rstudio-server.deb
    sudo sh -c 'apt install -y --no-install-recommends wget gdebi-core'
    wget -O "$DEB" https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2024.12.0-467-amd64.deb
    
    sudo mkdir -p /etc/rstudio
    sudo sh -c 'echo "auth-none=1" > /etc/rstudio/rserver.conf'
    sudo sh -c 'echo "USER=${rserver_user}" > /etc/default/rstudio-server'

    sudo gdebi --non-interactive "$DEB"
    rm "$DEB"

    # add script to .profile that adds coder environment vars in rstudio terminal
    echo "if [ -n \"\$RSTUDIO\" ]; then while IFS=$'\n' read -r line; do export \"\$line\"; done < \$HOME/.config/rstudio/coder.env ;fi" >> $HOME/.profile
fi
