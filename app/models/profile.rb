# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  include ImageUploader::Attachment(:avatar)
  include SoftDeletable
  include TimelineOwnable

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/

  has_many :follows, dependent: :restrict_with_exception, foreign_key: :source_profile_id, inverse_of: :source_profile
  has_many :inverse_follows, class_name: "Follow", dependent: :restrict_with_exception, foreign_key: :target_profile_id, inverse_of: :target_profile
  has_many :followees, class_name: "Profile", source: :target_profile, through: :follows
  has_many :followers, class_name: "Profile", source: :source_profile, through: :inverse_follows
  has_many :posts, dependent: :restrict_with_exception

  enum :profilable_type, {user: 0, organization: 1}, prefix: :as

  validates :idname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  sig { override.returns(String) }
  def timeline_key
    "timeline:profile:#{id}"
  end

  sig { returns(ImageUploader::UploadedFile) }
  def master_avatar
    avatar[:master]
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def my_profile?(target_profile:)
    idname == target_profile.idname
  end

  T::Sig::WithoutRuntime.sig { returns(Post::PrivateRelation) }
  def home_timeline_posts
    post_ids = ::Timeline.new(self).get_post_ids
    Post.where(id: post_ids).preload(:profile).order(created_at: :desc)
  end

  sig { params(target_profile: Profile).returns(Profile::Friendship) }
  def follow(target_profile:)
    creator = Profile::Friendship.new(source_profile: self, target_profile:)

    return creator if creator.invalid?

    creator.follow
  end

  sig { params(target_profile: Profile).returns(Profile::Friendship) }
  def unfollow(target_profile:)
    creator = Profile::Friendship.new(source_profile: self, target_profile:)

    return creator if creator.invalid?

    creator.unfollow
  end

  sig { params(content: T.nilable(String)).returns(Post::Creator) }
  def new_post(content: nil)
    Post::Creator.new(profile: self, content:)
  end

  sig { params(post: Post).returns(Profile::Timeline::Home) }
  def add_post_to_home_timeline(post:)
    Profile::Timeline::Home.new(profile: self).add_post(post:)
  end
end
