/**
 * Static action registry.
 * Add new action modules here. Each module exports:
 *   { id, label, icon, priority, detect(content): boolean, Component }
 */

import math from './math.action'
import json from './json.action'
import url from './url.action'
import folder from './folder.action'
import base64 from './base64.action'
import unicode from './unicode.action'

const registry = [math, json, url, folder, base64, unicode]

const byId = Object.fromEntries(registry.map(a => [a.id, a]))

/** Get all registered actions. */
export function getAll() {
  return registry
}

/** Get a single action by id. */
export function getById(id) {
  return byId[id]
}

/** Default action_config for new installations. */
export function defaultConfig() {
  return Object.fromEntries(registry.map(a => [a.id, { enabled: true, priority: a.priority }]))
}
