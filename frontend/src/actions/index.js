import math from './math.action'
import json from './json.action'
import url from './url.action'
import folder from './folder.action'
import base64 from './base64.action'
import unicode from './unicode.action'
import curl from './curl.action'
import ws from './ws.action'
import urlDecode from './urldecode.action'

const registry = [math, json, url, folder, base64, unicode, curl, ws, urlDecode]

const byId = Object.fromEntries(registry.map(a => [a.id, a]))

export function getAll() {
  return registry
}

export function getById(id) {
  return byId[id]
}

export function defaultConfig() {
  return Object.fromEntries(registry.map(a => [a.id, { enabled: true, priority: a.priority }]))
}
