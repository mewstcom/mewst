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

  sig { returns(String) }
  def atname!
    T.must(atname)
  end

  sig { returns(String) }
  def avatar_url!
    T.must(avatar_url)
  end

  sig { returns(String) }
  def description!
    T.must(description)
  end

  sig { returns(String) }
  def name!
    T.must(name)
  end

  sig { returns(Profile) }
  def profile!
    T.must(profile)
  end

  sig { void }
  private def atname_uniqueness
    if Profile.where.not(id: profile!.id).find_by(atname:)
      errors.add(:atname, :atname_uniqueness)
    end
  end
end
