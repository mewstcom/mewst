# typed: strict
# frozen_string_literal: true

class SignUp
  include ActiveModel::Model
  include ActiveModel::Attributes

  attribute :email, :string
  attribute :idname, :string

  validates :email, email: true, presence: true
  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validate :email_uniqueness
  validate :idname_uniqueness

  private

  def email_uniqueness
    if Account.find_by(email:)
      errors.add(:email, :email_uniqueness)
    end
  end

  def idname_uniqueness
    if Profile.find_by(idname:)
      errors.add(:idname, :idname_uniqueness)
    end
  end
end
