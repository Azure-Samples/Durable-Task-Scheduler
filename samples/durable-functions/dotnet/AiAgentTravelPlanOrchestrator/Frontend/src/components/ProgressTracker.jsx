import React from 'react';

const ProgressTracker = ({ status }) => {
  if (!status) return null;

  // Default values
  const progress = status.progress || 0;
  const message = status.message || 'Processing your request...';
  const step = status.step || 'Starting';

  // Custom styling based on the current step
  const getStepColor = () => {
    switch (step) {
      case 'WaitingForApproval':
        return '#f39c12'; // amber
      case 'BookingTrip':
        return '#27ae60'; // green
      default:
        return '#3498db'; // blue
    }
  };

  return (
    <div className="progress-tracker">
      <div className="progress-bar-container">
        <div 
          className="progress-bar-fill" 
          style={{ 
            width: `${progress}%`,
            backgroundColor: getStepColor()
          }}
        />
      </div>
      
      <div className="progress-details">
        <h3>{message}</h3>
        {status.destination && (
          <p className="destination">Destination: <strong>{status.destination}</strong></p>
        )}
        {status.documentUrl && (
          <p className="document-link">
            <a href={status.documentUrl} target="_blank" rel="noopener noreferrer">
              View Travel Plan Document
            </a>
          </p>
        )}
      </div>
    </div>
  );
};

export default ProgressTracker;
