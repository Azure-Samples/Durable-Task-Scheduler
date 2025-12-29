import React from 'react';

const ProgressTracker = ({ status }) => {
  if (!status) return null;

  // Map step names to progress percentages and user-friendly messages
  const getStepInfo = (step) => {
    switch (step) {
      case 'GettingDestinations':
        return { progress: 20, message: '🌍 Finding perfect destinations...' };
      case 'CreatingItinerary':
        return { progress: 50, message: '📅 Creating your itinerary...' };
      case 'GettingLocalRecommendations':
        return { progress: 75, message: '🍽️ Getting local recommendations...' };
      case 'WaitingForApproval':
        return { progress: 100, message: '✅ Plan ready for your approval!' };
      case 'BookingTrip':
        return { progress: 100, message: '🎯 Booking your trip...' };
      case 'Completed':
        return { progress: 100, message: '🎉 Trip booked successfully!' };
      default:
        return { progress: 10, message: '🚀 Starting your travel plan...' };
    }
  };

  const stepInfo = getStepInfo(status.step);
  const progress = status.progress || stepInfo.progress;
  const message = status.message || stepInfo.message;

  // Custom styling based on the current step
  const getStepColor = () => {
    switch (status.step) {
      case 'WaitingForApproval':
        return '#f39c12'; // amber
      case 'BookingTrip':
      case 'Completed':
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
