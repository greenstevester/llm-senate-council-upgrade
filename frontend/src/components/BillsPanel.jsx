import { useState, useEffect } from 'react';
import './BillsPanel.css';

export default function BillsPanel({ onClose }) {
  const [bills, setBills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState({ chamber: 'all', search: '' });
  const [lastUpdated, setLastUpdated] = useState(null);

  useEffect(() => {
    loadBills();
  }, []);

  const loadBills = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('http://localhost:8001/api/bills');
      if (!response.ok) throw new Error('Failed to fetch bills');
      const data = await response.json();
      setBills(data.bills || []);
      setLastUpdated(new Date(data.last_updated));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const filteredBills = bills.filter(bill => {
    const matchesChamber = filter.chamber === 'all' ||
                           bill.chamber.toLowerCase().includes(filter.chamber.toLowerCase());
    const matchesSearch = bill.title.toLowerCase().includes(filter.search.toLowerCase());
    return matchesChamber && matchesSearch;
  });

  return (
    <div className="bills-panel">
      <div className="bills-panel-header">
        <h2>Bills Before Parliament</h2>
        <button className="close-btn" onClick={onClose}>×</button>
      </div>

      {/* Filter controls */}
      <div className="bills-filters">
        <input
          type="text"
          placeholder="Search bills..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          className="search-input"
        />
        <select
          value={filter.chamber}
          onChange={(e) => setFilter({...filter, chamber: e.target.value})}
          className="chamber-select"
        >
          <option value="all">All Chambers</option>
          <option value="senate">Senate</option>
          <option value="house">House of Representatives</option>
        </select>
      </div>

      {/* Refresh button */}
      <div className="bills-actions">
        <button onClick={loadBills} disabled={loading} className="refresh-btn">
          {loading ? 'Loading...' : 'Refresh'}
        </button>
        {lastUpdated && (
          <span className="last-updated">
            Updated: {lastUpdated.toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* Bills list */}
      <div className="bills-list">
        {loading && <div className="loading">Loading bills...</div>}
        {error && (
          <div className="error">
            <p>Error: {error}</p>
            <button onClick={loadBills} className="retry-btn">Retry</button>
          </div>
        )}
        {!loading && !error && filteredBills.length === 0 && (
          <div className="no-bills">No bills found</div>
        )}
        {!loading && !error && filteredBills.map(bill => (
          <BillCard key={bill.id} bill={bill} />
        ))}
      </div>
    </div>
  );
}

function BillCard({ bill }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="bill-card">
      <div className="bill-header" onClick={() => setExpanded(!expanded)}>
        <h3>{bill.title}</h3>
        <span className="expand-icon">{expanded ? '▼' : '▶'}</span>
      </div>

      {(bill.chamber || bill.status) && (
        <div className="bill-meta">
          {bill.chamber && <span className="chamber-badge">{bill.chamber}</span>}
          {bill.status && <span className="status-badge">{bill.status}</span>}
        </div>
      )}

      {(bill.date_introduced || bill.portfolio_sponsor) && (
        <div className="bill-info">
          {bill.date_introduced && (
            <div className="info-row">
              <span className="label">Introduced:</span>
              <span>{bill.date_introduced}</span>
            </div>
          )}
          {bill.portfolio_sponsor && (
            <div className="info-row">
              <span className="label">Portfolio:</span>
              <span>{bill.portfolio_sponsor}</span>
            </div>
          )}
        </div>
      )}

      {expanded && (
        <div className="bill-details">
          {bill.summary && <p className="bill-summary">{bill.summary}</p>}
          <div className="bill-links">
            {bill.bill_url && (
              <a href={bill.bill_url} target="_blank" rel="noopener noreferrer">
                View Bill
              </a>
            )}
            {bill.explanatory_memo_url && (
              <a href={bill.explanatory_memo_url} target="_blank" rel="noopener noreferrer">
                Explanatory Memorandum
              </a>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
