interface ResponseErrorType extends Error {
  response: Response;
}

class ResponseError extends Error implements ResponseErrorType {
  response: Response;

  constructor(response: Response) {
    super(`Request failed with status code ${response.status}`);
    this.response = response;
    this.name = "ResponseError";
  }
}

interface RequestOptions extends RequestInit {
  headers?: Record<string, string>;
}

const request = async (url: string, method: string, options: RequestOptions = {}) => {
  const csrfToken = document.querySelector('meta[name="csrf-token"]')?.getAttribute("content");
  const headers = {
    ...(options.headers || {}),
    "Content-Type": "application/json",
    ...(csrfToken && { "X-CSRF-Token": csrfToken }),
  };

  delete options.headers;

  const res = await fetch(url, { ...{ method }, headers, ...options });

  if (!res.ok) {
    throw new ResponseError(res);
  }

  const data = await res.text();
  return data === "" ? {} : JSON.parse(data);
};

const requestWithData = async (
  url: string,
  method: string,
  data: Record<string, unknown> = {},
  options: RequestOptions = {},
) => {
  const body = JSON.stringify(data);

  return await request(url, method, {
    ...{ body },
    ...options,
  });
};

export default {
  get: async (url: string, options: RequestOptions = {}) => {
    return await request(url, "GET", { ...options });
  },

  post: async (url: string, data: Record<string, unknown> = {}, options: RequestOptions = {}) => {
    return await requestWithData(url, "POST", data, { ...options });
  },

  patch: async (url: string, data: Record<string, unknown> = {}, options: RequestOptions = {}) => {
    return await requestWithData(url, "PATCH", data, { ...options });
  },

  delete: async (url: string, data: Record<string, unknown> = {}, options: RequestOptions = {}) => {
    return await requestWithData(url, "DELETE", data, { ...options });
  },
};
