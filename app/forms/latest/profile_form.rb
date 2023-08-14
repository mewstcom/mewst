# typed: strict
# frozen_string_literal: true

class Latest::ProfileForm < Latest::ApplicationForm
  attribute :atname, :string
  attribute :avatar_url, :string
  attribute :description, :string
  attribute :name, :string

  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  validates :atname, format: {with: Profile::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :profile, presence: true
  validate :atname_uniqueness

  sig { void }
  private def atname_uniqueness
    if Profile.where.not(id: profile.not_nil!.id).find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end
end
