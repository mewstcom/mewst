# typed: strict
# frozen_string_literal: true

class StampChecker
  extend T::Sig

  sig { params(profile: T.nilable(ProfileRecord), posts: T.any(ActiveRecord::Relation, T::Array[PostRecord])).void }
  def initialize(profile:, posts:)
    @profile = profile
    @posts = posts
    @stamped_post_ids = T.let(nil, T.nilable(T::Array[T::Mewst::DatabaseId]))
  end

  sig { params(post: PostRecord).returns(T::Boolean) }
  def stamped?(post:)
    !profile.nil? && stamped_post_ids.include?(post.id)
  end

  sig { returns(T::Array[T::Mewst::DatabaseId]) }
  private def stamped_post_ids
    @stamped_post_ids ||= begin
      return [] if profile.nil? || posts.empty?

      profile.not_nil!.stamp_records.where(post_record: posts).pluck(:post_record_id)
    end
  end

  sig { returns(T.nilable(ProfileRecord)) }
  attr_reader :profile
  private :profile

  sig { returns(T.any(ActiveRecord::Relation, T::Array[PostRecord])) }
  attr_reader :posts
  private :posts
end
