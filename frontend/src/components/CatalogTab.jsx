import React, { useState, useEffect } from 'react';
import { Search, ShoppingBag, AlertCircle, RefreshCw } from 'lucide-react';

export default function CatalogTab({ onFetchProducts }) {
  const [products, setProducts] = useState([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [loading, setLoading] = useState(false);

  const loadData = async () => {
    setLoading(true);
    try {
      const res = await onFetchProducts(statusFilter, search);
      setProducts(res.items || []);
      setTotal(res.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [statusFilter]);

  const handleSearchSubmit = (e) => {
    e.preventDefault();
    loadData();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem', flexWrap: 'wrap', gap: '0.75rem' }}>
        <div>
          <h3 style={{ fontSize: '1.1rem', fontWeight: 700 }}>Local Store Catalog</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
            Current catalog state stored in the shop's database ({total} products)
          </p>
        </div>

        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <form onSubmit={handleSearchSubmit} style={{ display: 'flex', gap: '0.5rem' }}>
            <input
              type="text"
              className="input-text"
              placeholder="Search by SKU or name..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <button type="submit" className="btn btn-secondary">
              <Search size={16} />
            </button>
          </form>

          <select
            className="select-custom"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="ALL">All Statuses</option>
            <option value="ACTIVE">Active for Sale</option>
            <option value="WITHDRAWN">Withdrawn from Sale</option>
          </select>

          <button className="btn btn-secondary" onClick={loadData} disabled={loading} title="Refresh catalog">
            <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {products.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '3.5rem 1rem', color: 'var(--text-muted)' }}>
          <ShoppingBag size={36} color="var(--text-muted)" style={{ margin: '0 auto 0.75rem' }} />
          <p style={{ fontWeight: 600, color: 'var(--text-primary)' }}>No products in local catalog</p>
          <p style={{ fontSize: '0.85rem' }}>Start a catalog update above to populate the local inventory.</p>
        </div>
      ) : (
        <div className="table-responsive">
          <table>
            <thead>
              <tr>
                <th>SKU</th>
                <th>Product Name & Description</th>
                <th>Price</th>
                <th>Catalog Status</th>
                <th>Supplier Version Time</th>
                <th>Last Local Update</th>
              </tr>
            </thead>
            <tbody>
              {products.map((p) => (
                <tr key={p.id}>
                  <td>
                    <span className="code-badge" style={{ fontWeight: 700, color: '#f8fafc' }}>
                      {p.sku}
                    </span>
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, color: '#f8fafc' }}>{p.name}</div>
                    <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)', marginTop: '0.15rem' }}>
                      {p.description || 'No description provided'}
                    </div>
                  </td>
                  <td>
                    <span style={{ fontWeight: 700, fontFamily: 'var(--font-mono)', color: '#38bdf8' }}>
                      ${(p.price_cents / 100).toFixed(2)}
                    </span>
                  </td>
                  <td>
                    {p.status === 'ACTIVE' ? (
                      <span className="badge badge-completed">Active</span>
                    ) : (
                      <span className="badge badge-withdrawn">Withdrawn</span>
                    )}
                  </td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    {new Date(p.supplier_updated_at).toLocaleString()}
                  </td>
                  <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    {new Date(p.updated_at).toLocaleTimeString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
