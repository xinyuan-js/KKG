/** @type {import('next').NextConfig} */
const blogAPI = process.env.API_BASE_SERVER || "http://host.docker.internal:8080";
const ojAPI = process.env.OJ_API_BASE_SERVER || "http://host.docker.internal:8121";

const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: "/blog-api/:path*",
        destination: `${blogAPI}/:path*`
      },
      {
        source: "/oj-api/:path*",
        destination: `${ojAPI}/:path*`
      }
    ];
  }
};

export default nextConfig;
