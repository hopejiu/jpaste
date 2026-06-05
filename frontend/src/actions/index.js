import math from './math.action'
import json from './json.action'
import url from './url.action'
import folder from './folder.action'
import base64 from './base64.action'
import unicode from './unicode.action'

const registry = [math, json, url, folder, base64, unicode]

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
