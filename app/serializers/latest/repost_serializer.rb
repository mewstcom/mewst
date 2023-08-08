# typed: false
# frozen_string_literal: true

class Latest::RepostSerializer < Latest::ApplicationSerializer
  has_one :target_post, serializer: Latest::PostSerializer
  has_one :target_profile, serializer: Latest::ProfileSerializer
  has_one :original_post, serializer: Latest::PostSerializer
  has_one :original_profile, serializer: Latest::ProfileSerializer
end
