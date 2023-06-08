# typed: strict
# frozen_string_literal: true

class Forms::Profile < Forms::Base
  attribute :atname, :string
  attribute :avatar_url, :string
  attribute :description, :string
  attribute :name, :string

  sig { returns(Profile) }
  attr_accessor :profile

  validates :atname, format: {with: Profile::ATNAME_FORMAT}, length: {maximum: 30}, presence: true
  validates :profile, presence: true
  validate :atname_uniqueness

  private

  sig { void }
  def atname_uniqueness
    if Profile.where.not(id: profile.id).find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end
end
