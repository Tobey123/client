// @flow
import {globalColors} from '../styles'

const followingStateToStyle = (followingState: Constants.FollowingState) => {
  return {
    Following: {
      color: globalColors.green2,
    },
    NoState: {},
    NotFollowing: {
      color: globalColors.blue,
    },
    You: {
      fontStyle: 'italic',
    },
  }[followingState]
}

export {
  followingStateToStyle
}
