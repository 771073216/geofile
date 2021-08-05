#!/usr/bin/env bash
CHINA_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/accelerated-domains.china.conf"
GOOGLE_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/google.china.conf"
APPLE_DOMAINS_URL="https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/apple.china.conf"

findinclude() {
  b=$(awk -F':' '/include:/ {print$2}' out)
  sed -i '/include:/d' out
  for list in $b; do
    cat data/"$list" >> out
  done
  if grep include: out > /dev/null; then
    findinclude
  fi
}

setup() {
  a=$(awk -F':' '/include:/ {print$2}' data/"$1")
  for list in $a; do
    cat data/"$list" >> out
  done
}

build_cn(){
  setup cn
  findinclude
  curl -sSL $CHINA_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
  curl -sSL $GOOGLE_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
  curl -sSL $APPLE_DOMAINS_URL | perl -ne '/^server=\/([^\/]+)\// && print "$1\n"' >> out
  sed -i '/^#/d' out
  sed '/^$/d' out | sort --ignore-case -u > own/cn
  rm out
}

build_ad(){
  setup category-ads
  findinclude
  sed -i '/^#/d' out
  sed '/^$/d' out | sort --ignore-case -u > own/category-ads
  rm out
}

build_ad_all(){
  setup category-ads-all
  findinclude
  sed -i '/^#/d' out
  sed '/^$/d' out | sort --ignore-case -u > own/category-ads-all
  rm out
}

main() {
  cd community || exit 1
  mkdir own
  build_cn
  build_ad_all
  build_ad
  go run ./ --datapath=own/ --outputname=geosite.dat --outputdir=../publish/
  cd ..
  rm -rf community
}

main
