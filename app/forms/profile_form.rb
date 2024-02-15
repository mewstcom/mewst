# typed: strict
# frozen_string_literal: true

class ProfileForm < ApplicationForm
  attribute :atname, :string
  attribute :avatar_url, :string
  attribute :description, :string
  attribute :name, :string

  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  validates :atname, format: {with: Profile::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :avatar_url, url: {allow_blank: true}
  validates :viewer, presence: true
  validate :atname_uniqueness

  sig { void }
  private def atname_uniqueness
    if Profile.where.not(id: viewer.not_nil!.profile_id).find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end
end
