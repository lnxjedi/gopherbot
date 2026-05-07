/**
 * Native synchronous HTTP module for Gopherbot JavaScript extensions.
 *
 * The runtime provides this as require("http"). This file exists so editors can
 * discover the API via JSDoc when configured to scan lib/.
 */

/**
 * @typedef {Object<string, string>} HttpHeaders
 */

/**
 * @typedef {Object<string, string>} HttpCookies
 */

/**
 * @typedef {Object<string, (string|number|boolean|Array<string|number|boolean>)>} HttpQuery
 */

/**
 * @typedef {Object} HttpAuth
 * @property {string} user
 * @property {string} pass
 */

/**
 * @typedef {Object} HttpRequestOptions
 * @property {string|HttpQuery} [query]
 * @property {HttpCookies} [cookies]
 * @property {HttpHeaders} [headers]
 * @property {string|Uint8Array|Array<number>|Object|Array<any>} [body]
 * @property {string} [form]
 * @property {string|number} [timeout] Go-style duration string, or seconds as a number.
 * @property {number} [timeoutMs] Milliseconds; convenience alias.
 * @property {HttpAuth} [auth]
 */

/**
 * @typedef {Object} HttpResponse
 * @property {string} body
 * @property {number} bodySize
 * @property {HttpHeaders} headers
 * @property {HttpCookies} cookies
 * @property {number} statusCode
 * @property {string} statusText
 * @property {boolean} ok
 * @property {string} url
 * @property {any|null} json Parsed when Content-Type is application/json or +json; null otherwise or on parse failure.
 */

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function get(url, options) {}

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function del(url, options) {}

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function head(url, options) {}

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function patch(url, options) {}

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function post(url, options) {}

/**
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function put(url, options) {}

/**
 * @param {string} method
 * @param {string} url
 * @param {HttpRequestOptions} [options]
 * @returns {HttpResponse}
 */
function request(method, url, options) {}

module.exports = {
  get,
  delete: del,
  head,
  patch,
  post,
  put,
  request,
};
