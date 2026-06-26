ARG ES_VERSION=8.14.3
FROM docker.elastic.co/elasticsearch/elasticsearch:${ES_VERSION}

ARG ES_VERSION
RUN elasticsearch-plugin install --batch "https://release.infinilabs.com/analysis-ik/stable/elasticsearch-analysis-ik-${ES_VERSION}.zip"
