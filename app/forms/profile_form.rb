# typed: strict
# frozen_string_literal: true

class ProfileForm < ApplicationForm
  attribute :atname, :string
  attribute :name, :string
  attribute :description, :string
  attribute :avatar_kind, :string
  attribute :gravatar_email, :string
  attribute :image_url, :string

  sig { returns(T.nilable(ActorRecord)) }
  attr_accessor :viewer

  validates :atname, format: {with: ProfileRecord::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :avatar_kind, inclusion: ProfileRecord.avatar_kinds.keys, presence: true
  validates :image_url, url: {allow_blank: true}
  validates :viewer, presence: true
  validate :atname_uniqueness
  validate :gravatar_email_required
  validate :image_url_required

  sig { void }
  private def atname_uniqueness
    if ProfileRecord.where.not(id: viewer.not_nil!.profile_id).find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end

  sig { void }
  private def gravatar_email_required
    if avatar_kind == ProfileAvatarKind::Gravatar.serialize && gravatar_email.blank?
      errors.add(:gravatar_email, :required)
    end
  end

  sig { void }
  private def image_url_required
    if avatar_kind == ProfileAvatarKind::External.serialize && image_url.blank?
      errors.add(:image_url, :required)
    end
  end
end
