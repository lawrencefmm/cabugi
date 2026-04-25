import type { NextConfig } from "next";

const apiProxyTarget = process.env.API_PROXY_TARGET ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/healthz",
        destination: `${apiProxyTarget}/healthz`,
      },
      {
        source: "/api/openapi/:path*",
        destination: `${apiProxyTarget}/openapi/:path*`,
      },
      {
        source: "/api/v1/:path*",
        destination: `${apiProxyTarget}/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
