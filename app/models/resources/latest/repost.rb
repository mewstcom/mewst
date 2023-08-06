# typed: false
# frozen_string_literal: true

class Resources::Latest::Repost < Resources::Latest::Base
  root_key :repost, :reposts

  one :target_post, resource: Resources::Latest::Post
  one :target_profile, resource: Resources::Latest::Profile
  one :original_post, resource: Resources::Latest::Post
  one :original_profile, resource: Resources::Latest::Profile
end
