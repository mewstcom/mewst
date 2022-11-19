# typed: strict
# frozen_string_literal: true

class SignUp
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :user, T.nilable(User)
    const :errors, T::Array[Error]
  end

  attribute :email, :string
  attribute :idname, :string

  validates :email, email: true, presence: true
  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validate :email_uniqueness
  validate :idname_uniqueness

  def create
    if invalid?
      return Result.new(user: nil, errors: errors.map {|error| Error.new(message: error.full_message)})
    end

    account = Account.create!(email:)
    user = account.users.create!
    user.create_profile!(idname:)

    Result.new(user:, errors: [])
  end

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
