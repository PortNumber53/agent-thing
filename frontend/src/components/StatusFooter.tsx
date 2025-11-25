import type { DockerStatus } from './TopNav'
import './StatusFooter.css'

type StatusFooterProps = {
  dockerStatus: DockerStatus
  dockerDetails: string
  dockerMessage: string
  websocketStatus: string
  currentTime: string
}

export function StatusFooter({
  dockerStatus,
  dockerDetails,
  dockerMessage,
  websocketStatus,
  currentTime,
}: StatusFooterProps) {
  return (
    <footer className='status-footer' role='status' aria-live='polite'>
      <div className='status-footer__left'>
        <div className='status-footer__item'>
          <span className='status-footer__label'>Docker:</span>
          <span className={`status-footer__value status-footer__value--${dockerStatus}`}>{dockerStatus}</span>
          {dockerDetails && <span className='status-footer__details'>{dockerDetails}</span>}
        </div>
        <div className='status-footer__item'>
          <span className='status-footer__label'>WS:</span>
          <span className='status-footer__value'>{websocketStatus}</span>
        </div>
        {dockerMessage && <div className='status-footer__message'>{dockerMessage}</div>}
      </div>
      <div className='status-footer__right'>
        <span className='status-footer__label'>Time:</span>
        <span className='status-footer__value'>{currentTime}</span>
      </div>
    </footer>
  )
}

export default StatusFooter
