/** @type {import('next').NextConfig} */
const blogAPI = process.env.API_BASE_SERVER || "http://host.docker.internal:8080";

const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: "/blog-api/:path*",
        destination: `${blogAPI}/:path*`
      }
    ];
  }
};

export default nextConfig;
