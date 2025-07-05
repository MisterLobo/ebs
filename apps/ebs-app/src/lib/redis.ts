import { Redis } from 'ioredis'

let redisClient: Redis | undefined
export const getRedisClient = () => {
  if (redisClient) {
    return redisClient
  }
  redisClient = new Redis(process.env.REDIS_HOST ?? '')
  return redisClient
}