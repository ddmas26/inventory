import { Routes, Route, Navigate } from 'react-router-dom';
import AppLayout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Products from './pages/Products';
import Inventories from './pages/Inventories';
import StockPage from './pages/Stock';

export default function App() {
  return (
    <AppLayout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/products" element={<Products />} />
        <Route path="/inventories" element={<Inventories />} />
        <Route path="/stock" element={<StockPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppLayout>
  );
}
