//@ts-check

// eslint-disable-next-line @typescript-eslint/no-var-requires
const { composePlugins, withNx } = require('@nx/next');

/**
 * @type {import('@nx/next/plugins/with-nx').WithNxOptions}
 **/
const nextConfig = {
  env: {
    FBASE_API_KEY: process.env.FBASE_API_KEY,
    FBASE_AUTH_DOMAIN: process.env.FBASE_AUTH_DOMAIN,
    FBASE_PROJECT_ID: process.env.FBASE_PROJECT_ID,
    FBASE_STORAGE_BUCKET: process.env.FBASE_STORAGE_BUCKET,
    FBASE_MESSAGING_SENDER_ID: process.env.FBASE_MESSAGING_SENDER_ID,
    FBASE_APP_ID: process.env.FBASE_APP_ID,
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
