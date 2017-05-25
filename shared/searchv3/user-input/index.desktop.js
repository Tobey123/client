// @flow
import * as Constants from '../../constants/searchv3'
import React, {Component} from 'react'
import {AutosizeInput, Box, Text, Icon, ClickableBox} from '../../common-adapters'
import {globalColors, globalMargins, globalStyles} from '../../styles'
import IconOrAvatar from '../icon-or-avatar'
import {followingStateToStyle} from '../shared'
import {getStyle as getTextStyle} from '../../common-adapters/text'

import type {IconType} from '../../common-adapters/icon'
import type {Children} from 'react'

export type UserItemProps = {|
  followingState: Constants.FollowingState,
  icon: IconType,
  service: Constants.Service,
  username: string,
|}

export type Props = {|
  placeholder: string,
  userItems: Array<UserItemProps>,
  defaultUsernameText: string,
  onChangeText: (usernameText: string) => void,
|}

const UserItem = ({followingState, icon, service, username}: Props) => {
  return (
    <Box style={_pillStyle}>
      <IconOrAvatar icon={icon} service={service} username={username} avatarSize={16}
        style={{
          fontSize: 16,
          // Add more space to the left of square icons
          marginLeft: service === 'Hacker News' || service === 'Facebook' ? 3 : 0,
        }}
      />
      <Text
        type="BodySemibold"
        style={{
          ...followingStateToStyle(followingState),
          lineHeight: '18px',
          marginLeft: globalMargins.xtiny,
          marginBottom: 2,
        }}
      >{username}</Text>
      <Icon
        type="iconfont-close"
        style={{fontSize: 12, marginLeft: globalMargins.tiny, cursor: 'pointer'}}
      />
    </Box>
  )
}

const _pillStyle = {
  ...globalStyles.flexBoxRow,
  ...globalStyles.flexBoxCenter,
  height: 24,
  paddingLeft: globalMargins.xtiny,
  // 2 pixel fudge to accomodate built-in padding to iconfont-close
  paddingRight: globalMargins.tiny - 2,
  paddingTop: globalMargins.xtiny,
  paddingBottom: globalMargins.xtiny,
  marginRight: globalMargins.xtiny,
  marginTop: 2,
  marginBottom: 2,
  borderRadius: 24,
  borderWidth: 1,
  borderStyle: 'solid',
  borderColor: globalColors.black_10,
}

const _inputStyle = {
  ...getTextStyle('Body'),
  color: globalColors.black_75,
  border: 'none',
  outline: 'none',
  lineHeight: '22px',
  paddingLeft: 2,
  paddingRight: 0,
  paddingTop: 0,
  paddingBottom: 0,
  marginBottom: 2,
}

class UserInput extends Component<void, Props, void> {
  //callbacks
  //clickable areas
  //+ button
  state = {
    usernameText: this.props.defaultUsernameText,
  }

  _onChangeText = (usernameText) => {
    this.setState({usernameText})
  }

  render() {
    const {placeholder, userItems, usernameText, onChangeText} = this.props
    return (
      <Box style={{...globalStyles.flexBoxRow, alignItems: 'center', flexWrap: 'wrap'}}>
        {userItems.map(item => <UserItem {...item} key={item.username} />)}
        <AutosizeInput style={_inputStyle} placeholder={placeholder} value={usernameText} onChange={onChangeText}  />
      </Box>
    )
  }
}

export default UserInput
