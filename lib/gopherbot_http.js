/**
 * gopherbot_http.js
 *
 * Thin wrapper for the native "gopherbot_http" module so editors can
 * discover the API via JSDoc when configured to scan lib/.
 */

/**
 * @typedef {Object<string, string>} HttpHeaders
 */

/**
 * @typedef {Object<string, (string|number|boolean|Array<string|number|boolean>)>} HttpQuery
 */

/**
 * @typedef {Object} HttpClientOptions
 * @property {string} [baseURL]
 * @property {HttpHeaders} [headers]
 * @property {number} [timeoutMs]
 * @property {boolean} [throwOnHTTPError]
 */

/**
 * @typedef {Object} HttpRequestOptions
 * @property {string} [method]
 * @property {string} [path]
 * @property {string} [url]
 * @property {HttpQuery} [query]
 * @property {HttpHeaders} [headers]
 * @property {string|Uint8Array|Buffer} [body]
 * @property {number} [timeoutMs]
 * @property {boolean} [throwOnHTTPError]
 */

/**
 * @typedef {Object} HttpResponse
 * @property {number} status
 * @property {string} statusText
 * @property {Object<string, string[]>} headers
 * @property {string} body
 * @property {() => any} json
 */

/**
 * @typedef {Object} HttpClient
 * @property {(options: HttpRequestOptions) => HttpResponse} request
 * @property {(path: string, options?: HttpRequestOptions) => any} getJSON
 * @property {(path: string, payload: any, options?: HttpRequestOptions) => any} postJSON
 * @property {(path: string, payload: any, options?: HttpRequestOptions) => any} putJSON
 */

const native = require("gopherbot_http");

module.exports = {
  /**
   * Create a new HTTP client using the native module.
   * @param {HttpClientOptions} [options]
   * @returns {HttpClient}
   */
  createClient: native.createClient,
};
