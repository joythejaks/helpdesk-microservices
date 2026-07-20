class AppUser {
  const AppUser({
    required this.id,
    required this.email,
    required this.role,
    this.name = '',
    this.department = '',
    this.availability = 'offline',
  });

  final int id;
  final String email;
  final String role;
  final String name;
  final String department;
  final String availability;

  factory AppUser.fromJson(Map<String, dynamic> json) {
    return AppUser(
      id: json['id'] as int,
      email: json['email'] as String,
      role: json['role'] as String,
      name: (json['name'] as String?) ?? '',
      department: (json['department'] as String?) ?? '',
      availability: (json['availability'] as String?) ?? 'offline',
    );
  }
}
