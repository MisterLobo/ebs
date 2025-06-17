import { io, Socket } from 'socket.io-client'

let _socket: Socket | undefined

export const getSocket = () => {
  const host: string = process.env.NEXT_PUBLIC_API_HOST ?? ''
  if (!_socket) {
    _socket = io(`${host}/sub`, {
      retries: 10,
      reconnectionDelay: 10_000,
      reconnectionDelayMax: 10_000,
      auth: async () => ({
        token: 'token',
      }),
      // transports: ['websocket'],
    })
  }
  return _socket as Socket
}