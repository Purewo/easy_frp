import axios from 'axios';

export const controlApi = axios.create({
  baseURL: 'http://127.0.0.1:8080',
  headers: { 'Content-Type': 'application/json' },
});

export const clientApi = axios.create({
  baseURL: 'http://127.0.0.1:7410',
  headers: { 'Content-Type': 'application/json' },
});
