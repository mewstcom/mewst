# typed: strict
# frozen_string_literal: true

class Profile < ApplicationRecord
  extend Enumerize

  include Postable
  include SoftDeletable
  include TimelineOwnable
  include Followable
  T.unsafe(self).include ImageUploader::Attachment(:avatar)

  IDNAME_FORMAT = /\A[A-Za-z0-9_]+\z/
  PROFILABLE_TYPE_ACCOUNT = :account
  PROFILABLE_TYPE_ORGANIZATION = :organization

  enumerize :locale, in: I18n.available_locales
  enumerize :profilable_type, in: [PROFILABLE_TYPE_ACCOUNT, PROFILABLE_TYPE_ORGANIZATION]

  validates :atname, format: {with: IDNAME_FORMAT}, length: {maximum: 20}, presence: true, uniqueness: true

  sig { override.returns(String) }
  def timeline_key
    "timeline:profile:#{id}"
  end

  sig { returns(T.nilable(ImageUploader::UploadedFile)) }
  def master_avatar
    avatar&.fetch(:master, nil)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def following?(target_profile:)
    follows.exists?(target_profile:)
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def me?(target_profile:)
    atname == target_profile.atname
  end

  T::Sig::WithoutRuntime.sig { returns(Post::PrivateRelation) }
  def home_timeline_posts
    post_ids = ::Timeline.new(self).get_post_ids
    Post.where(id: post_ids).preload(:profile).order(created_at: :desc)
  end

  sig { params(post: Post).returns(Profile::Timeline::Home) }
  def add_post_to_home_timeline(post:)
    Profile::Timeline::Home.new(profile: self).add_post(post:)
  end
end
