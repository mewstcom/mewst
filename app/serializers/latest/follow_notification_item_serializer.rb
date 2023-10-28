# typed: false
# frozen_string_literal: true

class Latest::FollowNotificationItemSerializer < Latest::ApplicationSerializer
  one :profile, resource: Latest::ProfileSerializer
end
