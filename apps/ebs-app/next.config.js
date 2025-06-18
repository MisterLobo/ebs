//@ts-check

// eslint-disable-next-line @typescript-eslint/no-var-requires
const { composePlugins, withNx } = require('@nx/next');

/**
 * @type {import('@nx/next/plugins/with-nx').WithNxOptions}
 **/
const nextConfig = {
  env: {
    NEXT_PUBLIC_FBASE_API_KEY: process.env.FBASE_API_KEY,
    NEXT_PUBLIC_FBASE_AUTH_DOMAIN: process.env.FBASE_AUTH_DOMAIN,
    NEXT_PUBLIC_FBASE_PROJECT_ID: process.env.FBASE_PROJECT_ID,
    NEXT_PUBLIC_FBASE_STORAGE_BUCKET: process.env.FBASE_STORAGE_BUCKET,
    NEXT_PUBLIC_FBASE_MESSAGING_SENDER_ID: process.env.FBASE_MESSAGING_SENDER_ID,
    NEXT_PUBLIC_FBASE_APP_ID: process.env.FBASE_APP_ID,
  },
  nx: {
    // Set this to true if you would like to use SVGR
    // See: https://github.com/gregberge/svgr
    svgr: false,
  },
  images: {
    remotePatterns: [
      {
        hostname: 'images.unsplash.com',
        protocol: 'https',
        pathname: '/**',
      }
    ],
  },
};

const plugins = [
  // Add more Next.js plugins to this list if needed.
  withNx,
];

module.exports = composePlugins(...plugins)(nextConfig);
