#!/usr/bin/env bash
CHINA_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/accelerated-domains.china.conf"
GOOGLE_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/google.china.conf"
APPLE_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/apple.china.conf"

findinclude(){
b=$(awk -F':' '/include:/ {print$2}' out)
sed -i '/include:/d' out
for list in $b; do
  cat data/"$list" >> out
done
if grep include: out > /dev/null; then
  findinclude
fi
}

main(){
cd community || exit 1
a=$(awk -F':' '/include:/ {print$2}' data/cn)
for list in $a; do
  cat data/"$list" >> out
done
findinclude
curl -sSL $CHINA_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
curl -sSL $GOOGLE_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
curl -sSL $APPLE_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
sed -i '/^#/d' out
sed '/^$/d' out | sort --ignore-case -u > cn
go run ./ --datapath=cn --outputname=geosite.dat --outputdir=../
cd ..
rm -rf community
}

main
