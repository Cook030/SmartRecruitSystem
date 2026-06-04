import { useState, useEffect, useRef } from 'react';
import { chatApi } from '../api';

interface Message {
  id?: number;
  role: string;
  content: string;
  created_at?: string;
  createdAt?: string;
}

export default function Chat() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    chatApi.history().then((res) => {
      setMessages(res.data.data.messages || []);
    });
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || loading) return;
    const userMsg = input.trim();
    setInput('');
    setMessages((prev) => [...prev, { role: 'user', content: userMsg }]);
    setLoading(true);

    try {
      const res = await chatApi.send(userMsg);
      const reply = res.data.data.reply;
      setMessages((prev) => [...prev, { role: 'assistant', content: reply }]);
    } catch (err: any) {
      setMessages((prev) => [...prev, { role: 'assistant', content: '请求失败: ' + (err.response?.data?.message || err.message) }]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ flex: 1, overflowY: 'auto', padding: 16, background: '#f5f5f5', borderRadius: 8, marginBottom: 16 }}>
        {messages.length === 0 && (
          <div style={{ textAlign: 'center', color: '#999', marginTop: 40 }}>
            <p>你好，我是招聘助手 AI。你可以问我：</p>
            <p>- 目前有多少人投递？</p>
            <p>- 哪个岗位最热门？</p>
            <p>- 有哪些候选人掌握 Go 语言？</p>
          </div>
        )}

        {messages.map((msg, idx) => (
          <div key={idx} style={{
            display: 'flex',
            justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
            marginBottom: 12,
          }}>
            <div style={{
              maxWidth: '70%',
              padding: '10px 16px',
              borderRadius: 12,
              background: msg.role === 'user' ? '#1677ff' : '#fff',
              color: msg.role === 'user' ? '#fff' : '#333',
              boxShadow: '0 1px 2px rgba(0,0,0,0.1)',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}>
              {msg.content}
            </div>
          </div>
        ))}

        {loading && (
          <div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 12 }}>
            <div style={{ padding: '10px 16px', borderRadius: 12, background: '#fff', color: '#999' }}>
              AI 思考中...
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div style={{ display: 'flex', gap: 8 }}>
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="输入你的问题..."
          style={{
            flex: 1, padding: 12, borderRadius: 8, border: '1px solid #d9d9d9',
            resize: 'none', height: 48, fontFamily: 'inherit', fontSize: 14,
          }}
        />
        <button
          onClick={handleSend}
          disabled={loading || !input.trim()}
          style={{
            padding: '0 24px', background: '#1677ff', color: '#fff',
            border: 'none', borderRadius: 8, cursor: 'pointer', fontSize: 14,
          }}
        >
          发送
        </button>
      </div>
    </div>
  );
}
