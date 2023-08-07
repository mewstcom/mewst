# typed: false
# frozen_string_literal: true

class Latest::Resources::Repost < Latest::Resources::Base
  root_key :repost, :reposts

  one :target_post, resource: Latest::Resources::Post
  one :target_profile, resource: Latest::Resources::Profile
  one :original_post, resource: Latest::Resources::Post
  one :original_profile, resource: Latest::Resources::Profile
end
