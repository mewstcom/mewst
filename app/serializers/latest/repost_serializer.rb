# typed: false
# frozen_string_literal: true

class Latest::RepostSerializer < Latest::ApplicationSerializer
  root_key :repost, :reposts

  one :target_post, resource: Latest::PostSerializer
  one :target_profile, resource: Latest::ProfileSerializer
  one :original_post, resource: Latest::PostSerializer
  one :original_profile, resource: Latest::ProfileSerializer
end
