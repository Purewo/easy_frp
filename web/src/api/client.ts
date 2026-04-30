import axios from 'axios';

export const CONTROL_SERVER_URL = 'http://149.118.158.112:18080';

export const controlApi = axios.create({
  baseURL: CONTROL_SERVER_URL,
  headers: { 'Content-Type': 'application/json' },
});

export const clientApi = axios.create({
  baseURL: 'http://127.0.0.1:7410',
  headers: { 'Content-Type': 'application/json' },
});
