// @flow
import * as Constants from '../constants/gregor'
import * as CommonConstants from '../constants/common'
import {keyBy} from 'lodash'
import {ReachabilityReachable} from '../constants/types/flow-types'
import {onFollowNotifications} from '../actions/profile'

const initialState: Constants.State = {
  reachability: {reachable: ReachabilityReachable.unknown},
  seenMsgs: {},
  followMsgs: {},
}

export default function(
  state: Constants.State = initialState,
  action: Constants.GregorActions
): Constants.State {
  switch (action.type) {
    case CommonConstants.resetStore:
      return {...initialState}
    case Constants.updateSeenMsgs:
      if (!action.error) {
        const newMsgs: Constants.MsgMap = keyBy(action.payload.seenMsgs, m => m.md.msgID.toString('base64'))
        return {
          ...state,
          seenMsgs: {
            ...state.seenMsgs,
            ...newMsgs,
          },
        }
      }
      break
    case Constants.updateFollowMsgs:
      if (!action.error) {
        const newMsgs: Constants.MsgMap = keyBy(action.payload.followMsgs, m => m.md.msgID.toString('base64'))
        onFollowNotifications(newMsgs)
        return {
          ...state,
          followMsgs: {
            ...state.followMsgs,
            ...newMsgs,
          },
        }
      }
      break
    case Constants.updateReachability:
      const {reachability} = action.payload
      return {
        ...state,
        reachability,
      }
  }
  return state
}
