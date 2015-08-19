# Чат-сервер
Программа является сервером, работающим на TCP-сокете, к которому могут подключаться пользователи и использовать его 
в качестве простого чата в текстовом виде. Предполагается, что в качестве клиента для подключения используется telnet.

Чат поддерживает личные и общие сообщения, некоторые команды.

К серверу в качестве компаньона идет стандартный тестовый модуль, который эмулирует одновременное к нему обращение 
большого числа пользователей, которые, в свою очередь, активно обмениваются большим количеством случайных сообщений
(как общих, так и персональных).
